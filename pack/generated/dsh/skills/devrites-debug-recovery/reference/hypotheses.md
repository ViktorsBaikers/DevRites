# Ranked hypotheses (3-5, falsifiable)

Generate **3-5 ranked hypotheses** *before* testing any of them.
Single-hypothesis generation anchors on the first plausible idea.

**Check past root causes first.** Search `.devrites/archive/*/evidence.md` with
native file search for the failure's error text, symbol, and file names. A hit
whose recorded correct hypothesis matches enters the ranked list with its
recurrence noted; it is still tested, never assumed. No archive or no hit →
skip silently.

Each hypothesis must be **falsifiable**: state the prediction it makes.

> Format: *"If <X> is the cause, then <changing Y> will make the failure
> disappear / <changing Z> will make it worse."*

If you cannot state the prediction, the hypothesis is a vibe: discard or
sharpen it.

**Show the ranked list to the user before testing.** Domain knowledge re-ranks
instantly ("we just deployed a change to #3"), or they've already ruled out
hypotheses. Cheap checkpoint, big time saver. Don't block on it: proceed with
your ranking if the user is AFK.
