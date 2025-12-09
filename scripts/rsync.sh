#!/bin/bash

rsync -avh --exclude={docs,mobile,build,.*} ../pvz imac:~/pvz/