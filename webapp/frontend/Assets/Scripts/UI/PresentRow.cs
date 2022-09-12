using System;
using Data;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

public class PresentRow : MonoBehaviour
{
    [SerializeField] private Image _iconImage;
    [SerializeField] private TextMeshProUGUI _nameText;
    [SerializeField] private TextMeshProUGUI _messageText;
    [SerializeField] private Button _receiveButton;

    private UserPresent _present;
    private Action<UserPresent> _onReceiveButtonPressed;

    private void Start()
    {
        _receiveButton.onClick.AddListener(() => ReceiveAsync());
    }
    public void SetItem(UserPresent present, Action<UserPresent> onReceiveButtonPressed)
    {
        _present = present;
        _onReceiveButtonPressed = onReceiveButtonPressed;
        var item = StaticItemMaster.Items[present.itemId];
        _iconImage.sprite = item.LoadIcon();
        _nameText.text = item.name;
        _messageText.text = present.presentMessage;
    }

    private async void ReceiveAsync()
    {
        _onReceiveButtonPressed?.Invoke(_present);
;    }
}
