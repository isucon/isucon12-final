using UnityEngine;
using UnityEngine.UI;

public class Isu : MonoBehaviour
{
    [SerializeField] private Sprite[] _isuSprites;
    [SerializeField] private Image _isuImage;

    public void SetRandomIsu()
    {
        var i = Random.Range(0, _isuSprites.Length);
        _isuImage.sprite = _isuSprites[i];
    }
}
