# Hi notes demo

Open the URL printed at startup. The OS chooses an available local port.
The demo needs no database, media server, or external services.
Stop it with Ctrl-C.

The controls send real `hi.Notify` commands through `hi.Handler`.

Bursts send notes 150 ms apart. The counter does not count
the notes currently visible.
A full reload starts a fresh demo session.
