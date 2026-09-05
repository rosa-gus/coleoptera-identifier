def is_coleoptera(predicted_class_name: str, coleoptera_taxa: set) -> bool:
    """
    Verifica se a classe predita pertence à ordem Coleoptera.
    Faz uma verificação mais flexível para lidar com variações de nomeclatura.
    """
    
    predicted = predicted_class_name.lower()
    
    if predicted in coleoptera_taxa:
        return True
    
    coleoptera_keywords = {
        'beetle', 'ladybug', 'ladybird', 'weevil', 'scarab', 'curculio',
        'carabid', 'cerambycid', 'chrysomelid', 'coccinellid'
    }
    
    for keyword in coleoptera_keywords:
        if keyword in predicted:
            return True
        
    return False