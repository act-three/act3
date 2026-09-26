# Hi popovers and dialogs demo

Open the URL printed at startup. The OS chooses an available local port.
Use `-listen 127.0.0.1:4445` for a fixed address, or `-dark` for a dark theme.
The demo needs no database, media server, external services, or custom JavaScript.
Stop it with Ctrl-C. Reloading starts a fresh application session.

Every open flag lives in the Go application and changes through ordinary
messages. `Present` pairs each flag with its dismissal message.
Buttons attach menus with `.Menu` and centered help popovers with `.Popover`;
`.Dialog` presents content centered in the viewport while retaining its receiver.
Content titles supply accessible names.
Contrasting panels demonstrate local-theme popovers and root-theme dialogs.
All presentations use their built-in appearance.
