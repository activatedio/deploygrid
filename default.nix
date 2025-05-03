with import <nixpkgs> {};

stdenv.mkDerivation {

  name = "deploygrid";
  buildInputs = with pkgs; [
    go
    gnumake
    kind
    kubectl
    kubernetes-helm
  ];
  hardeningDisable = [ "fortify" ];
  shellHook = ''
    export GOPATH=$HOME/go
    export PATH=$PATH:$HOME/go/bin
  '';
}
