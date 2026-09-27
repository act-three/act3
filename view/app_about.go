package view

import (
	"cmp"
	"runtime/debug"
	"time"

	"ily.dev/act3/hi"
	"ily.dev/domi"
	"ily.dev/domi/html"

	"ily.dev/act3/buildinfo"
	. "ily.dev/act3/ui"
)

// AppAbout renders build, runtime, and dependency metadata for the
// running server: jj and link-time provenance from info, Go toolchain
// and module data straight from bi.
func AppAbout(info buildinfo.Info, bi *debug.BuildInfo) hi.View {
	return hi.ScrollView(hi.Vertical,
		hi.VStack(
			hi.Text("About").
				Title("About"),
			aboutRuntime(info),
			aboutBuild(info, bi),
			aboutCommits(info),
			aboutModules(bi),
			aboutSettings(bi),
		).
			Alignment(hi.Leading).
			Gap(32i).
			Padding(hi.Edges(16i)),
	)
}

func aboutRuntime(info buildinfo.Info) hi.View {
	return aboutSection("Runtime", aboutGrid(
		aboutRow("Started", domi.Text(formatTime(info.StartTime))),
	))
}

func aboutBuild(info buildinfo.Info, bi *debug.BuildInfo) hi.View {
	return aboutSection("Build", aboutGrid(
		aboutRow("Build Time", domi.Text(info.BuildTime)),
		aboutRow("Change ID", domi.Text(info.ChangeID)),
		aboutRow("Commit ID", domi.Text(info.CommitID)),
		aboutRow("Go Version", domi.Text(bi.GoVersion)),
	))
}

func aboutSettings(bi *debug.BuildInfo) hi.View {
	var rows []domi.Node
	for _, s := range bi.Settings {
		rows = append(rows, aboutRow(s.Key, domi.Text(s.Value)))
	}
	return aboutSection("Settings", aboutGrid(rows...))
}

func aboutCommits(info buildinfo.Info) hi.View {
	return aboutSection("Commits", Code(CodeNowrap, CodeSize2)(domi.Text(info.Log)))
}

func aboutModules(bi *debug.BuildInfo) hi.View {
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

func aboutSection(title string, body domi.Node) hi.View {
	return hi.VStack(
		hi.Text(title),
		hi.HTML(body).Class("v-html-block"),
	).
		Alignment(hi.Leading).
		Gap(12i)
}

func aboutGrid(rows ...domi.Node) domi.Node {
	return html.Div(Class("v-about-grid"))(rows...)
}

func aboutRow(key string, value domi.Node) domi.Node {
	return domi.Fragment(
		html.Div(Class("v-about-key"))(domi.Text(key)),
		html.Div()(value),
	)
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
