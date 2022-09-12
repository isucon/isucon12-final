using UnityEngine;
using UnityEngine.UI;

public class ScreenManager : SingletonMonobehaviour<ScreenManager>
{
    [SerializeField]
    private GameObject _window;
    
    [SerializeField]
    private Button _footerHomeButton;
    [SerializeField]
    private Button _footerItemButton;
    [SerializeField]
    private Button _footerPresentButton;
    [SerializeField]
    private Button _footerGachaButton;

    public enum WindowType
    {
        Home,
        Item,
        Present,
        Gacha,
    }
    
    private void Start()
    {
        _footerHomeButton.onClick.AddListener(() => TransitWindow(WindowType.Home));
        _footerItemButton.onClick.AddListener(() => TransitWindow(WindowType.Item));
        _footerPresentButton.onClick.AddListener(() => TransitWindow(WindowType.Present));
        _footerGachaButton.onClick.AddListener(() => TransitWindow(WindowType.Gacha));
    }

    public void TransitWindow(WindowType type)
    {
        for (int i = 0; i < _window.transform.childCount; i++)
        {
            Destroy(_window.transform.GetChild(i).gameObject);
        }

        if (type == WindowType.Home)
        {
            return;
        }

        var path = "Prefabs/Screen/Screen" + type;
        var prefab = Resources.Load(path);
        if (prefab == null)
        {
            Debug.Log("Prefab is missing: " + path);
            return;
        }
        GameObject.Instantiate(prefab, _window.transform);
    }
}
