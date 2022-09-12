using System;
using System.Security.Cryptography;
using System.Threading.Tasks;
using Data;
using TMPro;
using Unity.VisualScripting;
using UnityEngine;
using UnityEngine.UI;
using Random = System.Random;

public class HomeManager : MonoBehaviour
{
    [SerializeField]
    private Button _rewardButton;

    [SerializeField]
    private TextMeshProUGUI _isuCoinText;
    
    [SerializeField]
    private TextMeshProUGUI _isuRateText;
    
    [SerializeField]
    private TextMeshProUGUI _isuRewardText;

    [SerializeField] private ItemIcon _hammer1Icon;
    [SerializeField] private ItemIcon _hammer2Icon;
    [SerializeField] private ItemIcon _hammer3Icon;

    [SerializeField] private GameObject _isuPrefab;
    [SerializeField] private RectTransform _isuRoot;

    private int currentIsu;
    
    private async void Start()
    {
        await GameManager.HomeAsync();

        _rewardButton.onClick.AddListener(() => ReceiveReward());
        
        RefreshStatus();
    }

    private void Update()
    {
        RefreshStatus();
    }

    public async Task RefreshDeckAsync()
    {
        await GameManager.HomeAsync();
        
        RefreshStatus();
    }

    private void RefreshStatus()
    {
        _isuCoinText.text = $"{TextUtil.FormatShortText(GameManager.userData.user.isuCoin)}";
        _isuRateText.text = $"{TextUtil.FormatShortText(GameManager.userData.isuCoin.totalPerSec)}";
        
        var now = DateTimeOffset.Now.ToUnixTimeSeconds();
        var diffSeconds = now - GameManager.userData.isuCoin.refreshTime;
        var rewardCoin = diffSeconds * GameManager.userData.isuCoin.totalPerSec;
        _isuRewardText.text = $"{TextUtil.FormatShortText(rewardCoin)}";
        
        SetHammerIcon(_hammer1Icon, GameManager.userData.deck.card1);
        SetHammerIcon(_hammer2Icon, GameManager.userData.deck.card2);
        SetHammerIcon(_hammer3Icon, GameManager.userData.deck.card3);
        
        SetIsu();
    }

    private void SetHammerIcon(ItemIcon icon, UserCard card)
    {
        if (card == null)
        {
            icon.SetIcon(null);
            return;
        }

        var item = StaticItemMaster.Items[card.cardId];
        icon.SetIcon(item.LoadIcon());
    }

    private void SetIsu()
    {
        var isuCount = (int)(GameManager.userData.user.isuCoin / 10000);
        var diff = isuCount - currentIsu;
        if (diff > 0)
        {
            for (int i = 0; i < diff; i++)
            {
                var go = Instantiate(_isuPrefab, _isuRoot);
                var isu = go.GetComponent<Isu>();
                isu.SetRandomIsu();

                var randomX = UnityEngine.Random.Range(0f, 960f);
                var randomY = UnityEngine.Random.Range(0f, 500f);
                var randomScale = UnityEngine.Random.Range(0.5f, 1f);
                var transform = (RectTransform)go.transform;
                transform.position = new Vector3(randomX, randomY, 0f);
                transform.localScale = new Vector3(randomScale, randomScale);

            }
        }
        else if (diff < 0)
        {
            for (int i = _isuRoot.childCount - 1; i >= isuCount; i--)
            {
                Destroy(_isuRoot.GetChild(i).gameObject);
            }
        }

        currentIsu = isuCount;
    }

    private async void ReceiveReward()
    {
        if (GameManager.userData.userDeck == null)
        {
            Debug.LogWarning("deck is null");
        }
        
        await GameManager.apiClient.RewardAsync();
        RefreshStatus();
    }
}
