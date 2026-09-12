package hi

import (
	"context"
	"net/url"
	"slices"
	"testing"

	"ily.dev/domi"
)

type navigationApp struct {
	stubApp
	update func(context.Context, int) domi.Cmd[int]
}

func (a *navigationApp) Update(ctx context.Context, m int) domi.Cmd[int] {
	return a.update(ctx, m)
}

func (a *navigationApp) Subscriptions(context.Context) domi.Sub[int] { return nil }

func TestInstanceURLChange(t *testing.T) {
	ctx := t.Context()
	app := &navigationApp{}
	in := &instance[int, *navigationApp]{app: app, path: []string{"initial"}}
	want := []string{"next"}
	calls := 0
	app.update = func(gotCtx context.Context, m int) domi.Cmd[int] {
		calls++
		if gotCtx != ctx || m != 7 {
			t.Fatalf("Update received context %v and message %d", gotCtx, m)
		}
		if !slices.Equal(in.path, want) {
			t.Errorf("path before app Update = %q, want %q", in.path, want)
		}
		return nil
	}
	in.Update(ctx, msg[int]{appmsg: 7, path: []string{"next"}})
	in.Update(ctx, wrapMsg(7))
	if !slices.Equal(in.path, want) {
		t.Errorf("ordinary message changed instance path to %q", in.path)
	}
	app.preview = func(_ context.Context, _ *url.URL, render PreviewRenderer) Preview {
		return render("/preview", Text("preview"))
	}
	in.Preview(ctx, &url.URL{Path: "/preview"})
	if !slices.Equal(in.path, want) {
		t.Errorf("preview changed instance path to %q", in.path)
	}
	want = nil
	in.Update(ctx, msg[int]{appmsg: 7, path: []string{}})
	if calls != 3 {
		t.Errorf("app Update called %d times, want 3", calls)
	}
}
