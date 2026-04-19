# cccesl

this claude code statusline reads the session's transcript to display this session's cache expiry time.
it assumes that a given session only uses one cache TTL.

this effort was initiated when i thought claude code caches had a 5 minute TTL, but working on this
statusline showed me that it may be 5 minutes or 1 hour. for my use, it seems to always be 1 hour.

admittedly, that makes this tool a bit less relevant; i still care about my 1 hour cache window, 
but it's a far cry from thinking i have a 5 minute cache window.

the original vision was to have a live timer, but that doesn't seem possible ([shameless claudeslop writeup](./live_countdown_writeup.md))
