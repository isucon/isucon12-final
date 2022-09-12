using System.Collections;
using System.Collections.Generic;
using Data;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

public class ExpItemRow : MonoBehaviour
{
    [SerializeField] private Image _iconImage;
    [SerializeField] private TextMeshProUGUI _nameText;
    [SerializeField] private TextMeshProUGUI _countText;
    [SerializeField] private TextMeshProUGUI _useCountText;
    [SerializeField] private Button _minusButton;
    [SerializeField] private Button _plusButton;
    
    public UserItem Item { get; private set; }
    public int UseCount { get; private set; } = 0;

    private void Awake()
    {
        _nameText.text = "";
        _countText.text = "0";
        _useCountText.text = "0";
    }

    public void SetExpItem(UserItem userItem)
    {
        Item = userItem;
        
        var item = StaticItemMaster.Items[userItem.itemId];
        _iconImage.sprite = item.LoadIcon();
        _nameText.text = item.name;
        _countText.text = userItem.amount.ToString();
        _useCountText.text = UseCount.ToString();
        
        _minusButton.onClick.AddListener(() =>
        {
            if (UseCount > 0)
            {
                UseCount--;
                _useCountText.text = UseCount.ToString();
            }
        });
        _plusButton.onClick.AddListener(() =>
        {
            if (UseCount < Item.amount)
            {
                UseCount++;
                _useCountText.text = UseCount.ToString();
            }
        });
    }
}
