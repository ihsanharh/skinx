"""Entry point for ``python -m skinx``."""

from .cli import main

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\nGoodbye!")
