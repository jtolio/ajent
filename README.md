# ajent

this is my attempt at writing my own coding agent harness.

(1)

the biggest goal with this harness is constant session persistence.
you can't run this agent without specifying a persistence file for
the session. this allows you to stop the agent at any point, and 
pick back up by reopening the agent with the same session file.

```
usage:
  ajail ajent --api-key=<KEY> <your-session-file>
```

to make this work, this harness uses the provider-independent
[bellman](http://github.com/modfin/bellman) library, which has
great support for provider-independent session serialization.

the session files are in the "heredocs json lines" format, which
was my idea for the least complicated agent session serialization
format that's also kind of human readable. i wrote
[heredocs json lines documentation](https://pkg.go.dev/github.com/jtolio/ajent/hjl).

if you want to switch agent models, you can stop the agent and
restart it with the same session file with a new model. you can
also make copies and edit and fork, and all of the normal file
operations you might do.

i prefer this to a complex ui that has fork operations and so
on.

(2)

speaking of complex uis, a secondary goal for this agent is to
*not* be a TUI. we leave the shell alone. if you want command
editing history, you can run this with 
[rlwrap](https://github.com/hanslub42/rlwrap).

(3)

like [pi](https://mariozechner.at/posts/2025-11-30-pi-coding-agent/),
this harness has no concept of privileges or permissions. that is
something you should get from the sandbox you run this harness in.
i use [ajail](https://github.com/jtolio/ajail).

(4)

i've tried to be thoughtful about the ergonomics of the tools for
the llms. for instance, reading and editing files 
[uses hashlines](https://blog.can.ac/2026/02/12/the-harness-problem/).

(5)

there are still many things to do. tool calling output is not very friendly
or formatted, and the `web_fetch` and `web_search` tools are hard to do in a
provider independent way, and are currently in need of improvement.

## license

see LICENSE
