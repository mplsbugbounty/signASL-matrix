let
  lib = import <nixpkgs> {};
in

lib.buildGoModule rec {

  pname = "signASLbot";
  version = "";

  buildInputs = [
    lib.pkg-config
    lib.olm
  ];

  src = lib.fetchFromGitHub {
    url = "https://github.com/mplsbugbounty/signASL-matrix/archive/refs/heads/master.zip";
    owner = "mplsbugbounty";
    repo = pname;
    rev = "2f70fbdb93b3b5306bf82c589089ee034e4db5a3";
    sha256 = "1584na0qvnlmlr82brjs0xdb0znj9r2prbfark7scnivmfn3ncwk";
  };

  vendorHash = "sha256-5KL2O8GM34/dRLy5WbJFk5Ipj8l/1bR97N4iWphdv+E=";

  meta = with lib; {
    description = "";
    longDescription = ''
	It's a pretty basic little bot really!
    '';
    homepage = "https://github.com/mplsbugbounty/signASL-matrix";
    license = "";
    maintainers = with maintainers; [ symys ];
  };
}
