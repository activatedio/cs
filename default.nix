with import <nixpkgs> {};

stdenv.mkDerivation {
  name = "cs";
  buildInputs = with pkgs; [
    go
    gnumake
  ];
  hardeningDisable = [ "fortify" ];
}
