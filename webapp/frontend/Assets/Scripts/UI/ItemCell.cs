using System;
using Data;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

public class ItemCell : MonoBehaviour
{
    [SerializeField] private TextMeshProUGUI _nameText;
    [SerializeField] private Image _iconImage;
    [SerializeField] private TextMeshProUGUI _levelText;
    [SerializeField] private TextMeshProUGUI _amountText;
    [SerializeField] private Toggle _equipToggle;
    [SerializeField] private Button _enhanceButton;
    [SerializeField] private Button _useButton;
    [SerializeField] private TextMeshProUGUI _descriptionText;

    public bool IsEquipOn => _equipToggle.isOn;
    public void SetCard(UserCard card, Action onEquip, Action onEnhance)
    {
        var item = StaticItemMaster.Items[card.cardId];
        _nameText.text = item.name;
        _iconImage.sprite = item.LoadIcon();
        _levelText.text = $"レベル {card.level}";
        _equipToggle.onValueChanged.AddListener((_) => onEquip());
        _enhanceButton.onClick.AddListener(() => onEnhance());
    }
    public void SetItem(UserItem userItem, Action<UserItem> onUse)
    {
        var item = StaticItemMaster.Items[userItem.itemId];
        _nameText.text = item.name;
        _iconImage.sprite = item.LoadIcon();
        _amountText.text = userItem.amount.ToString();
        _descriptionText.text = item.description;
        _useButton.onClick.AddListener(() => onUse(userItem));
    }

    public void SetType(ItemManager.ItemTabType type)
    {
        _equipToggle.gameObject.SetActive(type == ItemManager.ItemTabType.Equip);
        _enhanceButton.gameObject.SetActive(type == ItemManager.ItemTabType.Enhance);
        _levelText.transform.parent.gameObject.SetActive(type is ItemManager.ItemTabType.Equip or ItemManager.ItemTabType.Enhance);
        _amountText.transform.parent.gameObject.SetActive(type is ItemManager.ItemTabType.Exp or ItemManager.ItemTabType.Timer);
        _descriptionText.gameObject.SetActive(type == ItemManager.ItemTabType.Exp);
        _useButton.gameObject.SetActive(type == ItemManager.ItemTabType.Timer);
    }
}
