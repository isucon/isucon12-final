using System;
using Data;
using UnityEngine;
using UnityEngine.UI;

public class GachaTableDialog : MonoBehaviour
{
    [SerializeField] private RectTransform _listRoot;
    [SerializeField] private GameObject _gachaRateRowPrefab;
    [SerializeField] private Button _closeButton;

    public void SetData(GachaItemMaster[] gachaItemMasters)
    {
        for (int i = 0; i < gachaItemMasters.Length; i++)
        {
            var gachaItemMaster = gachaItemMasters[i];
            var go = Instantiate(_gachaRateRowPrefab, _listRoot);
            var row = go.GetComponent<GachaRateRow>();
            row.SetText(gachaItemMaster);
            row.SetBackgroundColor(i % 2 == 0);
        }
    }

    public void SetOnClose(Action onClose)
    {
        _closeButton.onClick.AddListener(() => onClose());
    }
}
