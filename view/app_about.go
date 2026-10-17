package view

import (
	"cmp"
	"runtime/debug"
	"time"

	"ily.dev/domi"
	"ily.dev/domi/html"

	"ily.dev/act3/buildinfo"
	. "ily.dev/act3/hi"
	"ily.dev/act3/ui"
)

// AppAbout renders build, runtime, and dependency metadata for the
// running server: jj and link-time provenance from info, Go toolchain
// and module data straight from bi.
func AppAbout(info buildinfo.Info, bi *debug.BuildInfo) View {
	return ScrollView(Vertical,
		VStack(
			Text("About").
				Title("About"),
			aboutRuntime(info),
			aboutBuild(info, bi),
			aboutCommits(info),
			aboutModules(bi),
			aboutSettings(bi),
		).
			Alignment(Leading).
			Gap(32i).
			Padding(Edges(16i)),
	)
}

func aboutRuntime(info buildinfo.Info) View {
	return aboutSection("Runtime", aboutGrid(
		aboutRow("Started", domi.Text(formatTime(info.StartTime))),
	))
}

func aboutBuild(info buildinfo.Info, bi *debug.BuildInfo) View {
	return aboutSection("Build", aboutGrid(
		aboutRow("Build Time", domi.Text(info.BuildTime)),
		aboutRow("Change ID", domi.Text(info.ChangeID)),
		aboutRow("Commit ID", domi.Text(info.CommitID)),
		aboutRow("Go Version", domi.Text(bi.GoVersion)),
	))
}

func aboutSettings(bi *debug.BuildInfo) View {
	var rows []domi.Node
	for _, s := range bi.Settings {
		rows = append(rows, aboutRow(s.Key, domi.Text(s.Value)))
	}
	return aboutSection("Settings", aboutGrid(rows...))
}

func aboutCommits(info buildinfo.Info) View {
	return aboutSection("Commits", ui.Code(ui.CodeNowrap, ui.CodeSize2)(domi.Text(info.Log)))
}

func aboutModules(bi *debug.BuildInfo) View {
	rows := []domi.Node{aboutModuleRow(
		bi.Main.Path,
		cmp.Or(bi.Main.Version, "(devel)"),
	)}
	for _, d := range bi.Deps {
		rows = append(rows, aboutModuleRow(d.Path, d.Version))
	}
	return aboutSection("Modules", aboutGrid(rows...))
}

func aboutModuleRow(path, version string) domi.Node {
	return aboutRow(path, domi.Text(version))
}

func aboutSection(title string, body domi.Node) View {
	return VStack(
		Text(title),
		HTML(body).Class("v-html-block"),
	).
		Alignment(Leading).
		Gap(12i)
}

func aboutGrid(rows ...domi.Node) domi.Node {
	return html.Div(ui.Class("v-about-grid"))(rows...)
}

func aboutRow(key string, value domi.Node) domi.Node {
	return domi.Fragment(
		html.Div(ui.Class("v-about-key"))(domi.Text(key)),
		html.Div()(value),
	)
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
