#!/usr/bin/env bash

go build && mv gotype /usr/local/bin/

echo "alias type=gotype" >> ~/.bash_profile