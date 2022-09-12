using UnityEngine;

public abstract class SingletonMonobehaviour<T> : MonoBehaviour where T : SingletonMonobehaviour<T>
{
    private static T _instance;
    public static T Instance {
        get {
            if (_instance != null)
            {
                return _instance;
            }

            _instance = FindObjectOfType<T>();
            if (_instance != null)
            {
                return _instance;
            }

            var t = typeof(T);
            var go = new GameObject(t.Name);
            _instance = go.AddComponent(t) as T;
            return _instance;
        }
    }
    void Awake()
    {
        if (_instance != null && _instance != this) {
            Destroy(gameObject);
            return;
        }

        DontDestroyOnLoad(this);
        _instance = this as T;
    }

    void OnDestroy() {
        _instance = null;
    }
}
