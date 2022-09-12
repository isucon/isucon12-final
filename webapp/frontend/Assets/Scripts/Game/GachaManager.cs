using System.Collections;
using System.Collections.Generic;
using Network;
using System.Linq;
using System.Threading.Tasks;
using Data;
using JetBrains.Annotations;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

public class GachaManager : MonoBehaviour
{
    [SerializeField] private Canvas canvas;
    
    [SerializeField] private Button tab1Button;
    [SerializeField] private Button tab2Button;
    [SerializeField] private Button tab3Button;

    [SerializeField] private TextMeshProUGUI gachaTitleText;

    [SerializeField] private Image gachaPickupImage;
    [SerializeField] private TextMeshProUGUI gachaPickupNameText;
    [SerializeField] private TextMeshProUGUI gachaPickupDescriptionText;
    [SerializeField] private TextMeshProUGUI gachaPickupIsuRateText;

    [SerializeField] [NotNull] private Button gachaDraw1Button;
    [SerializeField] [NotNull] private Button gachaDraw10Button;
    [SerializeField] [NotNull] private Button gachaTableButton;

    private ListGachaResponse.GachaData[] datas;
    private int _tabIndex;

    async void Awake()
    {
        canvas.gameObject.SetActive(false);
        await RefreshAsync();
        canvas.gameObject.SetActive(true);

        var tabButtons = new[] { tab1Button, tab2Button, tab3Button };
        for (int i = 0; i < 3; i++)
        {
            var tabIndex = i; // capture
            tabButtons[tabIndex].onClick.AddListener(() => SelectTab(tabIndex));
            if (tabIndex >= datas.Length)
            {
                tabButtons[tabIndex].enabled = false;
                continue;
            }

            tabButtons[tabIndex].enabled = true;
        }
        
        gachaDraw1Button.onClick.AddListener(() => DrawGacha(1));
        gachaDraw10Button.onClick.AddListener(() => DrawGacha(10));
        gachaTableButton.onClick.AddListener(ShowGachaTable);
    }

    private async Task RefreshAsync()
    {
        var res = await GameManager.apiClient.ListGachaAsync();
        datas = res.gachas
            .OrderBy(data => data.gacha.displayOrder)
            .ToArray();
        SelectTab(_tabIndex);
    }

    private void SelectTab(int tabIndex)
    {
        _tabIndex = tabIndex;
        var gacha = datas[tabIndex].gacha;
        gachaTitleText.text = gacha.name;
        
        var items = datas[tabIndex].gachaItemList;
        var pickup = items.Where(i => i.itemType == (int)ItemType.Hammer)
            .OrderByDescending(i => i.weight)
            .Select(i => StaticItemMaster.Items[i.itemId])
            .Last();

        if (pickup == null)
        {
            Debug.LogWarning("No pickup for hammer");
            return;
        }

        // var startDate = TextUtil.FormatDateFromUnixTime(gacha.startAt);
        // var endDate = TextUtil.FormatDateFromUnixTime(gacha.endAt);
        gachaPickupNameText.text = pickup.name;
        gachaPickupDescriptionText.text = pickup.description;
        gachaPickupIsuRateText.text = $"{pickup.amount_per_sec}/sec";
        gachaPickupImage.sprite = pickup.LoadIcon();
    }

    private async void DrawGacha(int gachaCount)
    {
        var consumeCoin = 1000 * gachaCount;
        var res = await GameManager.apiClient.DrawGachaAsync(datas[_tabIndex].gacha.id, gachaCount);
        await RefreshAsync();
        DialogManager.Instance.ShowRewardDialog(res.presents);
    }

    private void ShowGachaTable()
    {
        var gacha = datas[_tabIndex];
        DialogManager.Instance.ShowGachaTableDialog(gacha.gachaItemList);
    }
}
