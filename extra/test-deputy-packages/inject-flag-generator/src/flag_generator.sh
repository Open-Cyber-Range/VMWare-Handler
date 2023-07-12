#!/bin/sh

FLAG_PATH=~/HowdyBankInventory.db

cat <<EOF >$FLAG_PATH
ID - Name - Doubloons # - Gold Bars #
0 - Bob Rectangle Jodhpurs - $(shuf -i 1-50 -n 1) - $(shuf -i 0-5 -n 1)
1 - Garold G. 'The Slick' Wilson, Jr. - $(shuf -i 1-50 -n 1) - $(shuf -i 0-500 -n 1)
2 - 'P☆' - 0 - 0
3 - Mr. Eugene Grabbler - $(shuf -i 100-500 -n 1) - $(shuf -i 50-1500 -n 1)

EOF

md5sum $FLAG_PATH
cat $FLAG_PATH
