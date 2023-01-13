#!/bin/sh

divisor=17
epoch_seconds=$(date +%s)
remainder=$((epoch_seconds%divisor))

if [ $remainder -eq 0 ]
then
  echo 1
else
  awk 'BEGIN{print '1/$remainder'}'
fi
