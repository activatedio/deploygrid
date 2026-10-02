with import <nixpkgs> {};

stdenv.mkDerivation {

  name = "deploygrid";
  buildInputs = with pkgs; [
    nodejs_22
    go
    gnumake
    kind
    kubectl
    kubernetes-helm
    kubebuilder
  ];
  hardeningDisable = [ "fortify" ];
  shellHook = ''
    export GOPATH=$HOME/go
    export PATH=$PATH:$HOME/go/bin
  '';
}
