with import <nixpkgs> {};

stdenv.mkDerivation {
  name = "config";
  buildInputs = with pkgs; [
    go
    gnumake
  ];
  hardeningDisable = [ "fortify" ];
}
