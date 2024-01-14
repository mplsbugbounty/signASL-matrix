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
    rev = "449b7259ea5b7d19f26e11aa1ab4b795b093fdcf";
    sha256 = "0nv9why8azpz1r4zg6h1314q14habnn1bahd0a8i8lqyv87zl3fd";
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
