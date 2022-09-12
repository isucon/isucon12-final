using System;
using TMPro;
using UnityEngine;
using UnityEngine.UI;

public class MessageDialog : MonoBehaviour
{
    [SerializeField] private TextMeshProUGUI _titleText;
    [SerializeField] private TextMeshProUGUI _messageText;
    [SerializeField] private Button _closeButton;

    private void Awake()
    {
        _titleText.text = "";
        _messageText.text = "";
    }
    
    public void SetText(string title, string message, Action onClose)
    {
        _titleText.text = title;
        _messageText.text = message;
        _closeButton.onClick.AddListener(() => onClose());
    }
}
