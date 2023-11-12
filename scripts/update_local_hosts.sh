#!/bin/bash
set -eu -o pipefail

hosts=
for i in `seq 1 1 5`;do
  h=isucon12-$i
  ip=`ssh $h ip a show dev ens5 | grep -oP '(?<=inet )192\.168\.0\.[0-9]+'`

  for j in 1 2 3 4 5 bench;do
    h2=isucon12-$j
    #ssh $h2 "echo $ip $h | sudo tee -a /etc/hosts"
    ssh $h2 "sudo sed -i 's/.*$h$/$ip     $h/' /etc/hosts"
  done
done
