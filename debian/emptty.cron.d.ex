#
# Regular cron jobs for the emptty package
#
0 4	* * *	root	[ -x /usr/bin/emptty_maintenance ] && /usr/bin/emptty_maintenance
