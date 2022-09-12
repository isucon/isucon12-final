using System;
using Data;
using Unity.VisualScripting;
using UnityEngine;
using UnityEngine.UI;

public class DialogManager : SingletonMonobehaviour<DialogManager>
{
    [SerializeField]
    private GameObject _dialog;
    [SerializeField]
    private GameObject _contents;

    public void ShowLoginBonus(UserPresent[] presents)
    {
        ShowRewardDialog(presents, true);
    }

    public void ShowRewardDialog(UserPresent[] presents, bool isLoginBonus = false)
    {
        var go = ShowDialog("Prefabs/Dialog/DialogReward");
        if (go == null)
        {
            return;
        }
        
        var dialog = go.GetComponent<RewardDialog>();
        dialog.SetData(presents);
        dialog.onClose = CloseDialog;
        if (isLoginBonus) dialog.SetLoginBonus();
    }

    public void ShowMessageDialog(string title, string message)
    {
        var go = ShowDialog("Prefabs/Dialog/DialogMessage");
        if (go == null)
        {
            return;
        }
        
        var dialog = go.GetComponent<MessageDialog>();
        dialog.SetText(title, message, CloseDialog);
    }

    public void ShowEnhanceDialog(UserCard card, UserItem[] items, Action<UserCard, UserItem[], EnhanceDialog> onEnhance)
    {
        var go = ShowDialog("Prefabs/Dialog/DialogEnhance");
        if (go == null)
        {
            return;
        }
        
        var dialog = go.GetComponent<EnhanceDialog>();
        dialog.SetCard(card, items, CloseDialog, onEnhance);
    }

    public void ShowGachaTableDialog(GachaItemMaster[] gachaItemMasters)
    {
        var go = ShowDialog("Prefabs/Dialog/DialogGachaTable");
        if (go == null)
        {
            return;
        }
        
        var dialog = go.GetComponent<GachaTableDialog>();
        dialog.SetData(gachaItemMasters);
        dialog.SetOnClose(CloseDialog);
    }

    private GameObject ShowDialog(string path)
    {
        for (int i = 0; i < _contents.transform.childCount; i++)
        {
            Destroy(_contents.transform.GetChild(i).gameObject);
        }

        var prefab = Resources.Load(path);
        if (prefab == null)
        {
            Debug.Log("Prefab is missing: " + path);
            return null;
        }
        var go = (GameObject)Instantiate(prefab, _contents.transform);
        var rect = (RectTransform)go.transform;
        rect.localScale = Vector3.one;
        rect.anchorMin = Vector2.zero;
        rect.anchorMax = Vector2.one;
        rect.offsetMin = Vector2.zero;
        rect.offsetMax = Vector2.zero;
        
        _dialog.SetActive(true);
        return go;
    }

    private void CloseDialog()
    {
        _dialog.SetActive(false);
    }
}
