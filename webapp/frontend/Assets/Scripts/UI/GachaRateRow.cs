using Data;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

public class GachaRateRow : MonoBehaviour
{
    [SerializeField] private TextMeshProUGUI _nameText;
    [SerializeField] private TextMeshProUGUI _rateText;
    
    [SerializeField] private Image _nameBaseImage;
    [SerializeField] private Image _rateBaseImage;

    private void Awake()
    {
        SetText("", "");
    }

    public void SetText(GachaItemMaster gachaItemMaster)
    {
        var itemMaster = StaticItemMaster.Items[gachaItemMaster.itemId];
        SetText(itemMaster.name, $"{gachaItemMaster.weight / 10000f * 100f:F3}%");
    }

    public void SetText(string name, string rate)
    {
        _nameText.text = name;
        _rateText.text = rate;
    }

    public void SetBackgroundColor(bool isOddRow)
    {
        var color = isOddRow ? new Color32(243, 221, 198, 255) : new Color32(215, 176, 135, 255);
        _nameBaseImage.color = color;
        _rateBaseImage.color = color;
    }
}
