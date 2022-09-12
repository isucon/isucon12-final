using System;
using UnityEngine.UI;

public struct TmpDisabler : IDisposable
{
    private Button _button;

    public TmpDisabler(Button button)
    {
        _button = button;
        button.interactable = false;
    }

    public void Dispose()
    {
        _button.interactable = true;
    }
}
