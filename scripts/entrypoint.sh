#!/bin/sh
# Container entrypoint. Just hands off to the divoom binary; the
# subcommand (default `serve` per the Dockerfile CMD) takes over PID 1.
#
# Earlier this script also ran `divoom push` on start to refresh the
# frame's baked backgrounds + fonts. That created an infinite loop —
# push ends by crash-restarting divoom_app on the device (to reload
# fonts), which makes the frame's :9000 API unreachable for ~30 s,
# during which `divoom serve` hits a connection-refused and crash-
# exits, Docker restarts the container, push runs again, repeat.
#
# The right way to refresh bgs daily is a cron-driven `divoom push`
# from outside the serve process — kept separate so it doesn't drag
# down the rendering loop. To run a push manually:
#
#     docker exec divoom-dashboard divoom push
#
exec /usr/local/bin/divoom "$@"
