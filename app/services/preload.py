"""Download and initialize model artifacts before serving traffic."""

from app.services.bioclip import identifier


def main() -> None:
    """Load the configured model and report completion."""

    identifier.load()
    print("BioCLIP model artifacts are ready.")


if __name__ == "__main__":
    main()
