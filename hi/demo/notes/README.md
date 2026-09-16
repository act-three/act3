# Hi notes demo

Open the URL printed at startup. The OS chooses an available local port.
The demo needs no database, media server, or external services.
Stop it with Ctrl-C.

The controls send real `hi.Notify` commands through `hi.Handler`:

- Add a single note, a burst of six, three different heights, or a long note.
- Schedule a burst three seconds ahead, then hover, focus, or touch the existing
  stack before it arrives to see new notes appear during interaction.
- Navigate between the two demo pages and use browser Back and Forward to try
  note preservation.

Bursts send notes 150 ms apart. The counter tracks notes sent and notes awaiting
delivery; it does not count the notes currently visible. A full reload starts a
fresh demo session.

Only the latest three active notes are visible, even when expanded. Older notes
keep their remaining time and can reappear when newer notes are dismissed.
Expanded notes keep their natural height up to 6rem (96 scaled pixels),
independent of the viewport height or note count. Content beyond the cap is clipped.

Hover or keyboard focus expands the stack and pauses its four-second timers.
Escape returns focus to the page; hover or touch expansion keeps the stack open.
On touch, tap to expand and tap outside to collapse.
Swipe down or use a dismiss button to remove a note.
Also try switching browser tabs and enabling your system's reduced-motion preference.
