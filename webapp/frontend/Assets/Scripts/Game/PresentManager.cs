using System.Linq;
using System.Threading.Tasks;
using Data;
using UnityEngine;
using UnityEngine.UI;

public class PresentManager : MonoBehaviour
{
    [SerializeField] private GameObject _presentRowPrefab;
    [SerializeField] private RectTransform _listRoot;
    [SerializeField] private Button _receiveAllButton;
    [SerializeField] private Button _closeButton;
    
    private int _index = 1;
    private UserPresent[] _presents;
    
    async void Start()
    {
        await RefreshAsync();
        
        _receiveAllButton.onClick.AddListener(() => ReceiveAllAsync());
        _closeButton.onClick.AddListener(() => Close());
    }

    private async Task RefreshAsync()
    {
        for (int i = 0; i < _listRoot.childCount; i++)
        {
            Destroy(_listRoot.GetChild(i).gameObject);
        }
        
        var res = await GameManager.apiClient.ListPresentAsync(_index);
        _presents = res.presents;

        for (int i = 0; i < _presents.Length; i++)
        {
            var row = Instantiate(_presentRowPrefab, _listRoot).GetComponent<PresentRow>();
            row.SetItem(res.presents[i], (p) => OnReceivedRowAsync(p));
        }
    }

    private async void OnReceivedRowAsync(UserPresent present)
    {
        await GameManager.apiClient.ReceivePresentAsync(new[] {present.id});
        await RefreshAsync();
    }

    private async void ReceiveAllAsync()
    {
        await GameManager.apiClient.ReceivePresentAsync(_presents.Select(p => p.id).ToArray());
        await GameManager.HomeAsync(); // プレゼント受け取っても所持コインの情報が返ってこないので、別に叩く
        await RefreshAsync();
    }

    private void Close()
    {
        ScreenManager.Instance.TransitWindow(ScreenManager.WindowType.Home);
    }
}
