Space UI for Windows

Run space-ui.exe and keep it running: it runs Xray, and Xray stops with it.

First start
  Open http://localhost:2053/ and log in as admin / admin.
  Change the username and the password at once (Settings).

Forgot the password
  Stop space-ui.exe, then run in this folder:
    space-ui.exe setting -username NEW_NAME -password NEW_PASSWORD
  and start space-ui.exe again.

Files
  space-ui.db   the panel's database, next to space-ui.exe
  bin\          Xray and its geo files

Not on Windows
  The space-ui menu and its install and update scripts are for Linux only.
  IP Limit does not ban: it needs Fail2ban, which runs on Linux.
