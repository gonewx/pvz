#!/bin/bash

rsync -avh --exclude={docs,mobile,build,.*} ../pvz3 imac:~/pvz/