#!/bin/sh

SLEEP_SECONDS=3

echo hello "$USER", going to sleep for $SLEEP_SECONDS seconds\
     && sleep $SLEEP_SECONDS \
     && echo good morning and farewell
