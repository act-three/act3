package hi

import "testing"

func TestTextTrimEnvironment(t *testing.T) {
	var got [4]environment
	Render(VStack(
		view(envProbe(&got[0])).TextTrim(TextCap),
		view(envProbe(&got[1])).Padding(Edges(4)),
		VStack(view(envProbe(&got[2])).TextTrim(TextEx)).TextTrim(TextTop),
		view(envProbe(&got[3])).TextTrim(TextCap).TextTrim(0),
	).TextTrim(TextLastBaseline))
	for i, want := range []TextEdgeSet{TextCap, TextLastBaseline, TextEx, TextCap} {
		if got[i].textTrim != want {
			t.Errorf("view %d: trim = %b, want %b", i, got[i].textTrim, want)
		}
	}
}
