#!/bin/bash

###############################################################################
#            __     ______  ____    ____  _____ _____ _   _ ____              #
#            \ \   / /  _ \/ ___|  / ___|| ____|_   _| | | |  _ \             #
#             \ \ / /| | | \___ \  \___ \|  _|   | | | | | | |_) |            #
#              \ V / | |_| |___) |  ___) | |___  | | | |_| |  __/             #
#               \_/  |____/|____/  |____/|_____| |_|  \___/|_|                #
###############################################################################

CONF_DIR="/etc/${PWD##*/}"
mkdir -p $CONF_DIR

set_conf() {
        var=$1
        filename=$2
        file="$CONF_DIR/$filename"
        while [ ! -s "$file" ]; do
                read -p "Type $var: " answer
                if [[ -n "$answer" ]]; then
                        echo "$answer" > "$file"
                        echo "${var^} saved to file '$file'"

                else
                        echo "No $var provided!"
                fi
        done
}

set_conf "token" "token.json"

###############################################################################
#              ____   ___ _____   ____  _____ _____ _   _ ____                #
#             | __ ) / _ \_   _| / ___|| ____|_   _| | | |  _ \               #
#             |  _ \| | | || |   \___ \|  _|   | | | | | | |_) |              #
#             | |_) | |_| || |    ___) | |___  | | | |_| |  __/               #
#             |____/ \___/ |_|   |____/|_____| |_|  \___/|_|                  #
###############################################################################

# set bin
cp $PWD/bin/mezvenkisto /usr/local/bin
# set service
cp $PWD/conf/mezvenkisto.service /etc/systemd/system
systemctl daemon-reload
systemctl enable mezvenkisto
systemctl restart mezvenkisto
