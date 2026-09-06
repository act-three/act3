package ui

import "testing"

func TestControlSizeEnvironment(t *testing.T) {
	var got [5]environment
	shared := base{envProbe(&got[0])}
	Render(VStack(
		shared.ControlSize(Mini),
		base{envProbe(&got[1])}.Padding(Edges(4)),
		VStack(base{envProbe(&got[2])}.ControlSize(Large)).ControlSize(Small),
		base{envProbe(&got[3])}.ControlSize(Mini).ControlSize(Large),
		base{envProbe(&got[4])}.Frame(Width(40)).ControlSize(Small),
	))
	for i, want := range []ControlSize{Mini, Regular, Large, Mini, Small} {
		if got[i].controlSize != want {
			t.Errorf("view %d: size = %v, want %v", i, got[i].controlSize, want)
		}
	}
	Render(shared)
	if got[0].controlSize != Regular {
		t.Errorf("modifying shared view changed its original size: %v", got[0].controlSize)
	}

	Render(VStack(shared.Padding(Edges(4))).ControlSize(Small))
	if got[0].controlSize != Small {
		t.Errorf("size did not survive stack and padding: %v", got[0].controlSize)
	}
}
