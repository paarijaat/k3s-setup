#!/bin/sh

set -ex

echo "Removing k3s"
/usr/local/bin/k3s-killall.sh || true
sleep 2
/usr/local/bin/k3s-uninstall.sh || true

sleep 5
echo "Downloading k3s-setup.sh"
curl -sfL https://get.k3s.io > k3s-setup.sh
chmod +x k3s-setup.sh

echo "Installing single node k3s cluster"
INSTALL_K3S_EXEC='--disable=traefik --disable metrics-server --write-kubeconfig-mode "0644"' ./k3s-setup.sh

echo "Copying k3s config file to ~/.kube/config.k3s"
cp /etc/rancher/k3s/k3s.yaml ~/.kube/config.k3s

echo "Overwriting ~/.kube/config file. Original file stored at ~/kube/config.bak"
cp ~/.kube/config ~/.kube/config.bak || true
cp ~/.kube/config.k3s ~/.kube/config
