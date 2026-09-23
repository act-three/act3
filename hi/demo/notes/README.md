# Hi notes demo

Open the URL printed at startup. The OS chooses an available local port.
The demo needs no database, media server, or external services.
Stop it with Ctrl-C.

Most buttons send `hi.Notify` commands through `hi.Handler`.
“Client-side note” uses `Hi.notify("client-note")` to display a template
registered by the initial `hi.RegisterNote` command.
It works across navigation and keeps its Go Undo action.

Bursts send notes 150 ms apart. The counter does not count
the notes currently visible or client-side notifications.
A full reload starts a fresh demo session.
