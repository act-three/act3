package hi

import "testing"

func TestButtonStyleEnvironment(t *testing.T) {
	var got [5]environment
	shared := base{envProbe(&got[0])}
	Render(VStack(
		shared.ButtonStyle(Subtle),
		base{envProbe(&got[1])}.Padding(Edges(4)),
		VStack(base{envProbe(&got[2])}.ButtonStyle(Prominent)).ButtonStyle(Destructive),
		base{envProbe(&got[3])}.ButtonStyle(Subtle).ButtonStyle(Prominent),
		base{envProbe(&got[4])}.Frame(Width(40)).ButtonStyle(Destructive),
	))
	for i, want := range []ButtonStyle{Subtle, Bordered, Prominent, Subtle, Destructive} {
		if got[i].buttonStyle != want {
			t.Errorf("view %d: style = %v, want %v", i, got[i].buttonStyle, want)
		}
	}
	Render(shared)
	if got[0].buttonStyle != Bordered {
		t.Errorf("modifying shared view changed its original style: %v", got[0].buttonStyle)
	}

	Render(VStack(shared.Padding(Edges(4))).ButtonStyle(Destructive))
	if got[0].buttonStyle != Destructive {
		t.Errorf("style did not survive stack and padding: %v", got[0].buttonStyle)
	}
}
