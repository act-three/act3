package stimulus

import "ily.dev/domi"

func Action(value ...string) domi.Attr { return domi.Name("data-action", value...) }

func Controller(value ...string) domi.Attr { return domi.Name("data-controller", value...) }

func init() {
	domi.RegisterCombining("data-action", " ")
}

func Target(controller, name string) domi.Attr {
	return domi.Name("data-"+controller+"-target", name)
}

func Value(controller, name string) func(...string) domi.Attr {
	return func(value ...string) domi.Attr {
		return domi.Name("data-"+controller+"-"+name+"-value", value...)
	}
}
