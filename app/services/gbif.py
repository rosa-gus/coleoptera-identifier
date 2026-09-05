# global biodiversity information facility (GBIF).
# https://gbif.org | https://techdocs.gbif.org/en/openapi/

import requests
from typing import Set

def get_coleoptera_taxa() -> Set[str]:
    """
    Busca taxóns da ordem Coleoptera no GBIF
    """
    try:
        res = requests.get(
            "https://api.gbif.org/v1/species/search",
            params={
                "q": "Coleoptera",
                "rank": "ORDER",
                "limit": 1
            },
            timeout=10
        )
        res.raise_for_status()
        
        coleoptera_key = res.json()["results"][0]["key"]
                
        # buscar todos os taxóns filhos (gênero, espécies)
        all_taxa = set()
        offset = 0
        limit = 100
        
        while True:
            res = requests.get(
                f"https://api.gbif.org/v1/species/{coleoptera_key}/children",
                params={
                    "limit": limit,
                    "offset": offset
                },
                timeout=10
            )
            res.raise_for_status()
            
            data = res.json()
            results = data.get("results", [])
            
            if not results:
                break
            
            for result in results:
                # add scientific name
                if "canonicalName" in result:
                    all_taxa.add(result["canonicalName"].lower())
                    
                # add english common name, if has
                if "vernacularName" in result:
                    all_taxa.add(result["vernacularName"].lower())
                
            offset += limit
            
            if offset >= 1000:
                break
            
        common_names = {
            "beetle", "ladybug", "ladybird", "ground beetle", "long-horned beetle",
            "leaf beetle", "dung beetle", "rhinoceros beetle", "weevil", "firefly",
            "stag beetle", "click beetle", "water beetle", "tiger beetle"
        }
        
        all_taxa.update(common_names)
        
        return all_taxa
    
    except Exception as e: 
        print(f"Erro ao buscar dados do GBIF: {e}")
        return get_fallback_coleoptera_classes()

def get_fallback_coleoptera_classes() -> Set[str]:
    """
    Lista de fallback com algumas classes de Coleoptera
    """
    return {
        "beetle", "ladybug", "ladybird", "ground beetle", "carabid",
        "long-horned beetle", "cerambycid", "leaf beetle", "chrysomelid",
        "dung beetle", "scarab", "rhinoceros beetle", "weevil", "curculio",
        "firefly", "lightning bug", "stag beetle", "rove beetle",
        "water beetle", "tiger beetle", "click beetle", "jewel beetle",
        "bark beetle", "darkling beetle", "blister beetle", "cucujid"
    }