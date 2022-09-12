using System.Collections;
using System.Collections.Generic;
using Data;
using UnityEngine;
using UnityEngine.UI;

public class ItemIcon : MonoBehaviour
{
    [SerializeField] private Image _iconImage;

    private void Awake()
    {
        SetIcon(null);
    }

    public void SetIcon(Sprite icon)
    {
        if (icon == null)
        {
            gameObject.SetActive(false);
            return;
        }
        
        gameObject.SetActive(true);
        _iconImage.sprite = icon;
    }
}
