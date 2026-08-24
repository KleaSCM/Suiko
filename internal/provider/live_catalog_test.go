//go:build live

// カタログから実際に取った "provider/model" 参照で生成できるか、そして
// "Model not found" にならないかを生サーバで確認するの。
//
// Author: KleaSCM
// Email: KleaSCM@gmail.com
package provider

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestLiveCatalogModelRef(t *testing.T) {
	Opts, Err := KuyuMashima(context.Background(), "http://127.0.0.1:4096")
	if Err != nil {
		t.Fatalf("KuyuMashima: %v", Err)
	}
	var Free string
	for _, O := range Opts {
		if strings.Contains(O.Value, "nemotron-3.5-lightning-free") {
			Free = O.Value
			break
		}
	}
	if Free == "" {
		t.Fatalf("free model not found in catalog: %v", Opts)
	}
	t.Logf("using model ref %q", Free)

	Oc := SorawoKamikoshi("http://127.0.0.1:4096")
	ctx, Cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer Cancel()
	Stream, Err := Oc.Complete(ctx, Request{
		Model:    Free,
		Messages: []Message{{Role: RoleSystem, Content: "Be concise."}, {Role: RoleUser, Content: "Say the word hello."}},
	})
	if Err != nil {
		t.Fatalf("Complete: %v", Err)
	}
	var Full string
	GotFailed := false
	for Ev := range Stream {
		if Ev.Failed {
			GotFailed = true
			t.Logf("FAILED: %s", Ev.Message)
		}
		if Ev.Delta != "" {
			Full += Ev.Delta
		}
	}
	if GotFailed {
		t.Fatalf("turn failed — model ref %q broken", Free)
	}
	if len(Full) == 0 {
		t.Fatalf("no text generated for model ref %q", Free)
	}
	t.Logf("OK model=%q textLen=%d", Free, len(Full))
}
