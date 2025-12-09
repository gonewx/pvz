#!/bin/bash

rsync -avh --exclude={docs,build,.*} ../pvz3 imac:~/pvz/