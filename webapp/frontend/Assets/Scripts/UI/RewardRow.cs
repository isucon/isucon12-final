using System.Collections;
using System.Collections.Generic;
using Data;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

public class RewardRow : MonoBehaviour
{
    [SerializeField] private Image _iconImage;
    [SerializeField] private TextMeshProUGUI _nameText;
    [SerializeField] private TextMeshProUGUI _countText;

    public void SetPresent(UserPresent present)
    {
        var item = StaticItemMaster.Items[present.itemId];
        _iconImage.sprite = item.LoadIcon();
        _nameText.text = item.name;
        _countText.text = $"x{present.amount}";
    }
}
