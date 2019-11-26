#!/bin/bash
sudo mkdir -p /opt/MyLab/ &&
sudo cp MyLab /opt/MyLab/ &&
sudo cp mylab.service /etc/systemd/system/ &&
sudo systemctl enable mylab.service &&
sudo systemctl start mylab.service

