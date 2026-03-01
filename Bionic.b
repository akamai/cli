language: generic

matrix:
  include:
    - os: linux
      dist: bionic
      sudo: required
      env:
        - TOX_ENV=py27
    - os: linux
      dist: bionic
      sudo: required
      env:
        - TOX_ENV=py37
    - os: osx
      env:
        - TOX_ENV=py27
    - os: osx
      env:
        - TOX_ENV=py37

install:
  - if [[ "$TOX_ENV" == "py27" ]] && [[ "$TRAVIS_OS_NAME" == "linux" ]]; then pyenv global 2.7; fi
  - if [[ "$TOX_ENV" == "py37" ]] && [[ "$TRAVIS_OS_NAME" == "linux" ]]; then pyenv global 3.7; fi
  - if [[ "$TRAVIS_OS_NAME" == "osx" ]] && [[ "$TOX_ENV" == "py27" ]]; then pyenv install 2.7.17; pyenv global 2.7.17; eval "$(pyenv init -)"; fi
  - if [[ "$TRAVIS_OS_NAME" == "osx" ]] && [[ "$TOX_ENV" == "py37" ]]; then pyenv install 3.7.5; pyenv global 3.7.5; eval "$(pyenv init -)"; fi
  - if [[ "$TRAVIS_OS_NAME" == "osx" ]]; then curl -fsSL https://bootstrap.pypa.io/get-pip.py | sudo python; fi
  - if [[ "$TRAVIS_OS_NAME" == "osx" ]]; then wget https://repo.anaconda.com/miniconda/Miniconda3-latest-MacOSX-x86_64.sh -O /tmp/miniconda.sh; bash /tmp/miniconda.sh -b -p /tmp/miniconda; export MINICONDA="/tmp/miniconda/bin/python"; else wget https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh -O /tmp/miniconda.sh; bash /tmp/miniconda.sh -b -p /tmp/miniconda; export MINICONDA="/tmp/miniconda/bin/python"; fi
  - python --version
  - pip install -U tox pytest
  - pip install -e .

script:
  - python --version
  - tox -e $TOX_ENV
  - make test

notifications:
  email: true

  slack:
    rooms:
      secure: GOUanPMgPnway2tjAblGgoI3FKJD4bM+1fzcpTi4Rutd58WOb9ejgEOTOtfPhrjuUfQt1SP04C5igMQa8o9fagmtI4PUleSC2b8aGivB18gyvMWH4uceHA645+ElulT45SPqkzTYeRsGjrjKD5O1D3JFk6lwFsfzQM5yZ2QSplpMvGG9IvGVfulNzCuSUa66fzzsSRgDWqAfc76s2o6YTF+/gngbxYgwjZp+dGuWOBhEvxEcKXAq4ohSCxPKzTqR87DzS+IMH/nt71hOFZyHRO+5sUDBIxiTyGCIesS9gYVrMgvjCFp8QHWYapj3E3CFaBC5XOE1r/DfNyHfvdqBn6zLwhZPtrflg3dfD8+uPbrwfYRihraUCdI0NiprJGYTSYyduJeg5MqHGD5saTQtpNAMZshxTTse7FcAYT7oYPGf2pxkAGkKfMZb/z3aAIDBQpcUIoWBuZURShFZ1qBmxtVaIiZ9Wm7fLvXXpmINn0xJNeCva57otv4RVWvPj3X5xhCFdshkUeaZndTZPF9RFoCf2FNIR9p4hKqVeBcvpEyMWg/BrczVeLy44NiLgzb8mBx87rU+M9Hxi6OUsCdpAJ4UgHtuQ6XeO0P380KSTDOULdOUuxvPcwoB3QyVxP8lxQNCCH+dFBbeRdQhbu7PYeTmKp0P7/omFiXEGdIycvY=
    on_failure: always
    on_success: change
