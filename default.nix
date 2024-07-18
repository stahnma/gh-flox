with import <nixpkgs> {};

let
  gh-flox = pkgs.stdenv.mkDerivation rec {
    pname = "gh-flox";
    version = "0.1.0"; # you might want to update this to match the actual version

    src = pkgs.fetchFromGitHub {
      owner = "stahnma";
      repo = "gh-flox";
      rev = "main"; # you can replace this with a specific commit or tag
      sha256 = "0kkf2mwvss6i1822npcs5kkh0a4hglkprx691ysjwjjdf1csid1q";
    };

    buildInputs = [ pkgs.go ];

    buildPhase = ''
      export GOPATH=$(mktemp -d)
      export GOCACHE=$(mktemp -d)
      mkdir -p $GOPATH/src/github.com/stahnma
      cp -r $src $GOPATH/src/github.com/stahnma/gh-flox
      cd $GOPATH/src/github.com/stahnma/gh-flox
      go build -o $TMPDIR/gh-flox
    '';

    installPhase = ''
      mkdir -p $out/bin
      cp $TMPDIR/gh-flox $out/bin/
    '';

    meta = with pkgs.lib; {
      description = "A tool for managing flox installations via GitHub";
      homepage = "https://github.com/stahnma/gh-flox";
      license = licenses.mit; # Update this if the license is different
      maintainers = with maintainers; [ ];
      platforms = platforms.unix;
    };
  };
in gh-flox

