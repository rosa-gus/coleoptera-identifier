# simple CLI to identify species name without using the REST API.

from rich import print
from rich.panel import Panel
from rich.prompt import Prompt
import validators

image_url = str(Prompt.ask("Insira o URL da imagem com o sujeito"))

#if not validators.url(image_url):
    #raise ValueError("A string estabelecida não se confere URL.")

panel = Panel(
            f"[bold]🔎 Verificado:[/bold] [url]{image_url}[/url]\netc",
            title="[green bold]🪲 Coleoptera Identifier Experiment[/green bold]", 
            title_align="left",
            padding=1
        )

print("\n", panel, "\n")
