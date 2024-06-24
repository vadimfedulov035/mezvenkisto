#!/bin/bash

# directory to store user info
CONF_DIR="/etc/mezvenkisto"
mkdir "$CONF_DIR"

# file to store user info
SECRET_FILE="$CONF_DIR/token.json"

####################################################################################################
#                      __     ______  ____    ____  _____ _____ _   _ ____                         #
#                      \ \   / /  _ \/ ___|  / ___|| ____|_   _| | | |  _ \                        #
#                       \ \ / /| | | \___ \  \___ \|  _|   | | | | | | |_) |                       #
#                        \ V / | |_| |___) |  ___) | |___  | | | |_| |  __/                        #
#                         \_/  |____/|____/  |____/|_____| |_|  \___/|_|                           #
####################################################################################################

figlet "VDS SETUP"

if [ ! -d "/root/vds-setup" ]; then
        mkdir "/root/vds-setup"
        git clone "git@github.com:vadimfedulov035/vds-setup.git" "/root/vds-setup"
fi

source "/root/vds-setup/vars.sh"
source "/root/vds-setup/sys.sh"

get_vars "secret"

set_vds

####################################################################################################
#                  _____ ____   ____   ___ _____   ____  _____ _____ _   _ ____                    #
#                 |_   _/ ___| | __ ) / _ \_   _| / ___|| ____|_   _| | | |  _ \                   #
#                   | || |  _  |  _ \| | | || |   \___ \|  _|   | | | | | | |_) |                  #
#                   | || |_| | | |_) | |_| || |    ___) | |___  | | | |_| |  __/                   #
#                   |_| \____| |____/ \___/ |_|   |____/|_____| |_|  \___/|_|                      #
####################################################################################################

figlet "TG BOT SETUP"

set_tg_bot() {
	# set bin
	cp /root/mezvenkisto/bin/mezvenkisto /usr/local/bin
	# set service
	cp /root/mezvenkisto/conf/mezvenkisto.service /etc/systemd/system
	systemctl daemon-reload
	systemctl enable mezvenkisto
	systemctl restart mezvenkisto
}

set_tg_bot
