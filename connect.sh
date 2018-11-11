#!/bin/sh

wsta -I -H "Origin: http://localhost:8080" ws://localhost:8080/ws '{"event":"pusher:subscribe","data":{"channel":"demo"}}'
