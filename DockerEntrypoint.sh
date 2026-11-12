#!/bin/sh

# Start fail2ban
[ $SPACE_UI_ENABLE_FAIL2BAN == "true" ] && fail2ban-client -x start

# Run space-ui
exec /app/space-ui
