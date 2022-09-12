using System;
using System.Linq;
using Network;
using UnityEngine;
using UnityEngine.UI;

public class TitleManager : MonoBehaviour
{
    [SerializeField]
    public Button startButton;

    private void Start() {
        startButton.onClick.AddListener(OnStartButton);
    }

    private void OnStartButton() {
        LoginSequenceAsync();
    }

    private async void LoginSequenceAsync()
    {
        using var disabler = new TmpDisabler(startButton);
        LoadConfigFromUrl();
        await GameManager.CreateUserAsync();
        SceneController.LoadScene(SceneType.Game);
    }

    private static void LoadConfigFromUrl()
    {
        try
        {
            var url = Application.absoluteURL;
            if (string.IsNullOrEmpty(url))
            {
                return;
            }
            
            var uri = new Uri(url);
            var unescapedQuery = uri.GetComponents(UriComponents.Query, UriFormat.Unescaped);
            var queries = unescapedQuery.Split('&')
                .Select(x => x.Split('='))
                .Where(x => x.Length == 2)
                .ToDictionary(x => x[0], x => x[1]);

            if (queries.TryGetValue("host", out var host))
            {
                Debug.Log($"Set host: {host}");
                GameManager.apiClient.Host = host;
            }
        }
        catch (Exception e)
        {
            Debug.LogException(e);
        }
    }
}
