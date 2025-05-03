with import <nixpkgs> {};

stdenv.mkDerivation {

  name = "deploygrid";
  buildInputs = with pkgs; [
    go
    gnumake
  ];
  hardeningDisable = [ "fortify" ];
  shellHook = ''
    export GOPATH=$HOME/go
    export PATH=$PATH:$HOME/go/bin
  '';
}
