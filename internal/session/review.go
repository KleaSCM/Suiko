/**
 * Pending Writes (Suiko Session) — モデルの書き込みを人間が審める扉ね。
 *
 * 既定では人間の拒否権が優先：モデルの書き戻しは即座にディスクへ行かず、
 * この待ち行列に積あがる。UI（または外部ホスト）が Accept で通し、Reject で
 * 捨てる。auto_accept_writes の世界では何も滞留しない — 判定は自動で承認側に
 * 倒れるわ。
 * REFERENCE(KleaSCM): SuikoDesign.md §8 human veto by default
 *
 * Author: KleaSCM
 * Email: KleaSCM@gmail.com
 */
package session

import "suiko/internal/world"

// 待ち行列の1件。Apply を呼べば書き込みが実行される — 判定後の実行も
// 同じ原子的経路を通るから、直接書きと挙動が変わらないの。
type PendingWrite struct {
	Id    int
	Desc  string // UI 表示用の一行
	Kind  string // add-entry | update-entry | log-event
	Turn  int
	apply func() world.Error
}

// 審査キュー本体。FIFO — 先に生まれた書き込みから裁かれるの。
type Review struct {
	Pending    []PendingWrite
	nextId     int
	AutoAccept bool
}

func NewReview(AutoAccept bool) *Review {
	return &Review{AutoAccept: AutoAccept, nextId: 1}
}

// Submit hands a write to the gate. With auto-accept on, it applies
// immediately and never queues — the caller still gets an id back.
// The write only touches disk through the same atomic path either way,
// so queued acceptance and direct application behave identically.
func (R *Review) Submit(Desc, Kind string, Turn int, Apply func() world.Error) (int, world.Error) {
	R.nextId++
	Item := PendingWrite{
		Id:    R.nextId,
		Desc:  Desc,
		Kind:  Kind,
		Turn:  Turn,
		apply: Apply,
	}
	if R.AutoAccept {
		return Item.Id, Item.apply()
	}
	R.Pending = append(R.Pending, Item)
	return Item.Id, world.Error{}
}

// 承認してディスクへ通すの。未知idはゼロエラーで静かに無視 — 二度押しに強い。
func (R *Review) Accept(Id int) world.Error {
	for I, Item := range R.Pending {
		if Item.Id != Id {
			continue
		}
		Err := Item.apply()
		R.Pending = append(R.Pending[:I], R.Pending[I+1:]...)
		return Err
	}
	return world.Error{}
}

// 拒否。ただ捨てるだけ — 履歴にも残らない、最初から無かったことになるの。
func (R *Review) Reject(Id int) {
	for I, Item := range R.Pending {
		if Item.Id != Id {
			continue
		}
		R.Pending = append(R.Pending[:I], R.Pending[I+1:]...)
		return
	}
}

func (R *Review) List() []PendingWrite {
	return append([]PendingWrite{}, R.Pending...)
}
