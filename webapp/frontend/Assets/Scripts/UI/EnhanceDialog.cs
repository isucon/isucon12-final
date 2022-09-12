using System;
using System.Collections;
using System.Collections.Generic;
using Data;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

public class EnhanceDialog : MonoBehaviour
{
    [SerializeField] private TextMeshProUGUI _levelText;
    [SerializeField] private TextMeshProUGUI _levelMaxText;
    [SerializeField] private TextMeshProUGUI _expText;
    [SerializeField] private TextMeshProUGUI _isuRateText;
    [SerializeField] private Image _expGaugeImage;

    [SerializeField] private RectTransform _listRoot;
    [SerializeField] private GameObject _expItemRowPrefab;
    
    [SerializeField] private Button _closeButton;
    [SerializeField] private Button _enhanceButton;

    private UserCard _userCard;
    private UserItem[] _items;
    private ExpItemRow[] _itemRows;

    private void Awake()
    {
        Reset();
    }
    
    public void Reset()
    {
        _levelText.text = "";
        _levelMaxText.text = "";
        _expText.text = "";
        _isuRateText.text = "";
        _expGaugeImage.fillAmount = 0f;
    }

    public void SetCard(UserCard userCard, UserItem[] items, Action onClose, Action<UserCard, UserItem[], EnhanceDialog> onEnhance)
    { 
        _closeButton.onClick.AddListener(() => onClose());
        _enhanceButton.onClick.AddListener(() => OnEnhance(onEnhance));

        UpdateData(userCard, items);
    }

    public void UpdateData(UserCard userCard, UserItem[] items)
    {
        _userCard = userCard;
        _items = items;
        var card = StaticItemMaster.Items[userCard.cardId];

        _levelText.text = userCard.level.ToString();
        _levelMaxText.text = card.max_level.ToString();
        _expText.text = $"{userCard.totalExp}";
        _isuRateText.text = $"+{userCard.amountPerSec}";
        _expGaugeImage.fillAmount = CalculateExpGauge(userCard, card);

        for (int i = 0; i < _listRoot.childCount; i++)
        {
            Destroy(_listRoot.GetChild(i).gameObject);
        }

        _itemRows = new ExpItemRow[_items.Length];
        for (int i = 0; i < _itemRows.Length; i++)
        {
            var item = _items[i];
            var go = Instantiate(_expItemRowPrefab, _listRoot);
            var row = go.GetComponent<ExpItemRow>();
            row.SetExpItem(item);
            _itemRows[i] = row;
        }
    }

    private void OnEnhance(Action<UserCard, UserItem[], EnhanceDialog> onEnhance)
    {
        // copy list with used amount
        var items = new UserItem[_itemRows.Length];
        for (int i = 0; i < _items.Length; i++)
        {
            var item = _items[i];
            items[i] = new UserItem()
            {
                id = item.id,
                itemId = item.itemId,
                itemType = item.itemType,
                amount = _itemRows[i].UseCount,
            };
        }

        onEnhance(_userCard, items, this);
    }

    private static float CalculateExpGauge(UserCard userCard, ItemMaster master)
    {
        var basePoint = master.base_exp_per_level;
        int nextLevelExp, currentLevelExp;
        if (userCard.level == 1)
        {
            nextLevelExp = basePoint;
            currentLevelExp = 0;
        }
        else
        {
            // この計算ロジックはサーバーと合わせる
            nextLevelExp = (int)(basePoint * Math.Pow(1.2, userCard.level - 1));
            currentLevelExp = (int)(basePoint * Math.Pow(1.2, userCard.level - 2));
        }

        var step = userCard.totalExp - currentLevelExp;
        var diff = nextLevelExp - currentLevelExp;
        return (float)step / diff;
    }
}
