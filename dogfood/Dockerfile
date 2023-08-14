ARG BRANCH=main
FROM nixos/nix

RUN echo "experimental-features = nix-command flakes" >> /etc/nix/nix.conf

RUN nix develop github:coder/coder#${BRANCH} --print-build-logs
