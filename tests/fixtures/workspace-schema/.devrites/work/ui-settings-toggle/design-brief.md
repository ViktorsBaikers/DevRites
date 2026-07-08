# Design Brief

## Design direction
Quiet settings surface with a clear switch, existing spacing tokens, and no hero treatment.

## States
| State | Requirement |
| --- | --- |
| default | Current value is visible. |
| loading | Save is in progress and control is disabled. |
| error | Previous value remains and error text is announced. |
| success | Saved confirmation is announced. |

## Interaction model
Mouse click, keyboard Space, and screen-reader label all operate the same control.
